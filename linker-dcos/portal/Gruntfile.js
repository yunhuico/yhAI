'use strict';

module.exports = function(grunt) {
	// load all grunt tasks
	require('matchdep').filterDev('grunt-*').forEach(grunt.loadNpmTasks);
	// configurable paths
	var moduleConfig = {
		src: 'portal-ui',
		test: 'test',
		dist: 'target/portal-ui',
		target: 'target',
		server: 'portal-server',
		deps: 'node_modules',
		artifact: 'linker-dcos-portal.zip'
	};

	grunt.initConfig({
		module: moduleConfig,
		pkg: grunt.file.readJSON('package.json'),
		clean: {
			dist: {
				files: [{
					dot: true,
					src: [
						'<%= module.target %>',
						'<%= module.artifact %>',
						'! <%= module.target %> /.git'
					]
				}]
			}
		},
		imagemin: {
			dist: {
				options: {
					optimizationLevel: 3 //定义 PNG 图片优化水平
				},
				files: [{
					expand: true,
					cwd: '<%= module.src %>/img',
					src: ['*.{png,ico,jpg,gif,jpeg}'], // 优化 img 目录下所有 png/jpg/jpeg 图片
					dest: '<%= module.dist %>/img'
				}]
			}
		},
		htmlmin: {
			dist: {
				options: {

				},
				files: [{
					expand: true,
					cwd: '<%= module.dist %>',
					src: ['templates/**/*.html'],
					dest: '<%= module.dist %>'
				}, {
					expand: true,
					cwd: '<%= module.dist %>',
					src: ['*.html'],
					dest: '<%= module.dist %>'
				}]
			}
		},
		// Put files not handled in other tasks here
		copy: {
			dist: {
				files: [{
					expand: true,
					dot: true,
					cwd: '',
					dest: '<%= module.target %>',
					src: [
						'<%= module.server %>/**/*',
						'package.json',
						'entrypoint.sh'
					]
				}, {
					expand: true,
					dot: true,
					cwd: '',
					dest: '<%= module.target %>',
					src: [		
					    'node_modules/**'
					]
				}]
			}
		},
				
		compress: {
			main: {
				options: {
					archive: '<%= module.artifact %>'
				},
				files: [ // path下的所有目录和文件
					{
						cwd: '<%= module.target %>',
						expand: true,
						src: [
							'**'
						],
						dest: ''
					}
				]
			}
		},
		requirejs: {
		  compile: {
		    options: {
		      appDir: "portal-ui",
              baseUrl: "js",
		      mainConfigFile: ["portal-ui/js/main.js","portal-ui/js/loginMain.js"],
		      dir: "<%= module.target %>/portal-ui",
		      optimize: "uglify2",
		      optimizeCss: "standard",
		      generateSourceMaps: true,
		      preserveLicenseComments: false,
		      removeCombined: true,
		      modules: [
			        {
			            name: "main"
			        },
			         {
			            name: "loginMain"
			        }
			   ]
		    }
		  }
		}
	});

	grunt.registerTask('build', [
		'requirejs',		
		'htmlmin',
		//'imagemin',
		'copy'
	]);

	grunt.registerTask('deploy', [
		'build',
		'compress'
	]);

	grunt.registerTask('default', [
		'clean',
		'build'
	]);

}